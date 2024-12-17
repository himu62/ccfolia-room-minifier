/// <reference types="vite/client" />

declare module "virtual:webcontainer-fstree" {
    const fileSystemTree: import("@webcontainer/api").FileSystemTree;
    export default fileSystemTree;
}
