import { createHash } from "node:crypto";

export const humanizeSize = (size: number): string => {
    if (size < 1024) {
        return `${size} bytes`;
    } else if (size < 1024 ** 2) {
        return `${(size / 1024).toFixed(1)} KB`;
    } else if (size < 1024 ** 3) {
        return `${(size / 1024 ** 2).toFixed(1)} MB`;
    } else {
        return `${(size / 1024 ** 3).toFixed(1)} GB`;
    }
};

export const hashFile = (buffer: Buffer): string => {
    const hash = createHash("sha256");
    hash.update(buffer);
    return hash.digest("hex");
};
