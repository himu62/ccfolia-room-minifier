import type { Context } from "hono";
import unzipper from "unzipper";
import archiver from "archiver";
import { hashFile, humanizeSize } from "./util";
import { processDataJSON, processImage } from "./process";
import { TempBuffer } from "./buffer";

export const handler = async (c: Context) => {
	const body = await c.req.parseBody();
	if (!body.file) {
		console.error("エラー！ ファイルが指定されていません");
		c.json({ message: "ファイルが指定されていません" }, 400);
		return;
	}

	try {
		const srcFile = body.file as File;
		const buffer = Buffer.from(await srcFile.arrayBuffer());
		const directory = await unzipper.Open.buffer(buffer);

		// oldname -> newname
		const filenamesMap: { [filepath: string]: string } = {};
		let originalDataJSON = "";

		const totalFilesCount = directory.files.length;
		console.log(`ファイル読み込み完了!! ファイル数: ${totalFilesCount}, ファイルサイズ: ${humanizeSize(buffer.length)}`);

		const archive = archiver("zip", { zlib: { level: 9 } });
		const destBuffer = new TempBuffer();
		archive.pipe(destBuffer);

		let readFilesCount = 0;

		for (const file of directory.files) {
			readFilesCount++;

			if (file.path === ".token") continue;
			if (file.path === "__data.json") {
				originalDataJSON = (await file.buffer()).toString();
				continue;
			}
			if (await shouldProcess(file)) {
				const processed = await processImage(await file.buffer());
				const compressedRate = Number.parseFloat((processed.length / file.uncompressedSize).toFixed(1));
				console.log(`処理済み [${readFilesCount}/${totalFilesCount}] (${humanizeSize(file.uncompressedSize)} -> ${humanizeSize(processed.length)}, ${compressedRate}%)`);

				const newFilepath = hashFile(processed) + ".webp";
				filenamesMap[file.path] = newFilepath;
				archive.append(processed, { name: newFilepath });
				continue;
			}
			archive.append(file.stream(), { name: file.path });
		}

		const newDataJSON = processDataJSON(originalDataJSON, filenamesMap);
		const newToken = "0." + hashFile(Buffer.from(newDataJSON));
		
		archive.append(newDataJSON, { name: "__data.json" });
		archive.append(newToken, { name: ".token" });
		await archive.finalize();
		const outputBuffer = destBuffer.getBuffer();

		const compressedRate = Number.parseFloat((outputBuffer.length / buffer.length).toFixed(1));
		console.log(`処理完了！ ファイルサイズ: ${humanizeSize(buffer.length)} -> ${humanizeSize(outputBuffer.length)} (${compressedRate}%)`);

		c.header("Content-Type", "application/zip");
		c.header("Content-Disposition", `attachment; filename="${srcFile.name.replace(/\.zip$/, "")}-minified.zip"`);
		c.body(outputBuffer);
	} catch (e) {
		console.error("エラー！", e);
		c.json({ message: "エラーが発生しました" }, 500);
	}
};

const shouldProcess = async (file: unzipper.File): Promise<boolean> => {
	if (await isAnimatedPNG(file)) {
		return false;
	}
	if (file.path.endsWith(".png") || file.path.endsWith(".jpg")) {
		return true;
	}
	return false;
};

const isAnimatedPNG = async (file: unzipper.File): Promise<boolean> => {
	// PNGファイルの構造
	// [signature(8bytes)][chunks...]
	// chunk: [chunkLength(4bytes)][chunkType(4bytes)][chunkData(chunkLength bytes)][CRC32(4bytes)]

	const buffer = await file.buffer();

	if (buffer.length < 8) {
		return false;
	}

	const signature = buffer.subarray(0, 8).toString("ascii");
	if (signature !== "\x89PNG\r\n\x1a\n") {
		return false;
	}

	let offset = 8;
	while (offset < buffer.length) {
		const chunkLength = buffer.readUInt32BE(offset);
		const chunkType = buffer.subarray(offset + 4, offset + 8).toString("ascii");
		if (chunkType === "acTL") {
			return true;
		}
		offset += chunkLength + 12;
	}

	return false;
};
