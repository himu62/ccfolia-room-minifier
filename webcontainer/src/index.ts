import { Hono } from "hono";
import { cors } from "hono/cors";
import { serve } from "@hono/node-server";
import { handler } from "./handler";

const app = new Hono();
app.use("*", cors());
app.get("/", (c) => c.text("CCFOLIAルームファイルのサイズ小さくするアプリ"));
app.post("/process", handler);
serve(app);
