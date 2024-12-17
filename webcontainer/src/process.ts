import sharp from "sharp";

export const processImage = async (buffer: Buffer): Promise<Buffer> => {
    return await sharp(buffer).webp({
        preset: "picture",
        quality: 75,
        effort: 6,
    }).toBuffer();
};

export const processDataJSON = (dataJSON: string, filenamesMap: {[filepath: string]: string}): string => {
    let replaced = dataJSON;
    for (const [oldname, newname] of Object.entries(filenamesMap)) {
        replaced = replaced.replace(new RegExp(oldname, "g"), newname);
    }

    const data = JSON.parse(replaced);
    if (!("resources" in data)) {
        throw new Error("__data.json が壊れています");
    }

    interface ResourceMap {
        [filename: string]: {
            type: string;
        };
    }
    const isResourceMap = (value: unknown): value is ResourceMap => {
        if (typeof value !== "object" || value === null) {
            return false;
        }
        const obj = value as {[filename: string]: unknown};
        for (const key in obj) {
            if (typeof obj[key] !== "object" || obj[key] === null) {
                return false;
            }
            const item = obj[key] as {type?: unknown};
            if (typeof item.type !== "string") {
                return false;
            }
        }
        return true;
    }
    if (!isResourceMap(data.resources)) {
        throw new Error("__data.json が壊れています");
    }

    for (const newname of Object.keys(data.resources as ResourceMap)) {
        if (newname in Object.values(filenamesMap)) {
            data.resources.newname.type = "image/webp";
        }
    }

    return JSON.stringify(data);
};
