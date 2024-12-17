// biome-ignore lint/style/useImportType: react JSX
import React, { useEffect, useRef, useState } from "react";
import { WebContainer } from "@webcontainer/api";
import fileSystemTree from "virtual:webcontainer-fstree";

const App: React.FC = () => {
	const [wcontainerReady, setWcontainerReady] = useState(false);
	const [processedFile, setProcessedFile] = useState<Blob | null>(null);
	const [processedFileName, setProcessedFileName] = useState("");
	const [processing, setProcessing] = useState(false);
	const [wcontainerUrl, setWcontainerUrl] = useState("");
	const fileInputRef = useRef<HTMLInputElement | null>(null);

	const init = async () => {
		try {
			const wcontainer = await WebContainer.boot();
			await wcontainer.mount(fileSystemTree);

			const installProcess = await wcontainer.spawn("npm", ["ci"]);
			if (await installProcess.exit !== 0) {
				throw new Error("Failed to install dependencies");
			}

			await wcontainer.spawn("npm", ["run", "start"]);

			wcontainer.on("server-ready", (_port: number, url: string) => {
				setWcontainerReady(true);
				setWcontainerUrl(url);
			});

			wcontainer.on("error", (error: { message: string }) => {
				console.error(error.message);
			});
		} catch (e) {
			console.error(e);
		}
	};
	useEffect(() => {
		init();
	});

	const handleFileSubmit = async () => {
		const file = fileInputRef.current?.files?.[0];
		if (!file || !wcontainerReady || !wcontainerUrl) return;

		setProcessing(true);

		const formData = new FormData();
		formData.append("file", file);

		const res = await fetch(`${wcontainerUrl}/process`, {
			method: "POST",
			body: formData,
		});

		setProcessing(false);
		setProcessedFile(await res.blob());
		setProcessedFileName(`${file.name}-minified.zip`);
	};

	const handleDownload = () => {
		if (!processedFile) return;
		const url = URL.createObjectURL(processedFile);
		const a = document.createElement("a");
		a.href = url;
		a.download = processedFileName;
		a.click();
		URL.revokeObjectURL(url);
	};

	return (
		<>
			<h1>CCFOLIAルームファイルのサイズ小さくするアプリ</h1>
			{!wcontainerReady && <p>アプリケーションを起動中...</p>}
			{wcontainerReady && (
				<>
					<input type="file" ref={fileInputRef} />
					<button type="button" onClick={() => handleFileSubmit()}>
						小さくする
					</button>
					<iframe src={wcontainerUrl} />
					{processedFile && (
						<button type="button" onClick={handleDownload} disabled={processing}>
							ダウンロード
						</button>
					)}
				</>
			)}
		</>
	);
};

export default App;
