import type { FileSystemTree } from "@webcontainer/api";
import fs from "node:fs";
import path from "node:path";   
import type { PluginOption } from "vite";

export const directoryToFileSystemTree = (dirpath: string): FileSystemTree => {
    const result: FileSystemTree = {};
    const entries = fs.readdirSync(dirpath, { withFileTypes: true });
    for (const entry of entries) {
        if (entry.name === "node_modules") {
            continue;
        }
        const fullpath = path.join(dirpath, entry.name);
        if (entry.isDirectory()) {
            result[entry.name] = {
                directory: directoryToFileSystemTree(fullpath),
            };
        } else {
            const contents = fs.readFileSync(fullpath, "utf8");
            result[entry.name] = { file: { contents } };
        }
    }
    return result as FileSystemTree;
};

export const fileSystemTreePlugin = (targetDir: string): PluginOption => {
    const VIRTUAL_MODULE_ID = "virtual:webcontainer-fstree";
    const RESOLVED_VIRTUAL_MODULE_ID = "\0" + VIRTUAL_MODULE_ID;

    return {
        name: "webcontainer-fstree",
        resolveId(id: string) {
            if (id === VIRTUAL_MODULE_ID) {
                return RESOLVED_VIRTUAL_MODULE_ID;
            }
        },
        load(id: string) {
            if (id === RESOLVED_VIRTUAL_MODULE_ID) {
                return `export default ${JSON.stringify(directoryToFileSystemTree(targetDir))}`;
            }
        },
    };
};
