import type { ClientMessage, ServerMessage } from "./types/ws.js";

export type MessageHandler = (msg: ServerMessage) => void;

export interface BikClientOptions {
  /** Called when the connection is established and game_state is received */
  onReady?: MessageHandler;
  onMessage?: MessageHandler;
  onClose?: () => void;
  onError?: (err: Event) => void;
  /** Provide a WebSocket constructor for Node.js environments (e.g. from 'ws') */
  WebSocketImpl?: typeof WebSocket;
}

export class BikClient {
  private ws: WebSocket | null = null;
  private url: string;
  private token: string;
  private opts: BikClientOptions;

  constructor(url: string, token: string, opts: BikClientOptions = {}) {
    this.url = url;
    this.token = token;
    this.opts = opts;
  }

  connect(): void {
    const WS = this.opts.WebSocketImpl ?? WebSocket;
    const fullUrl = `${this.url}?token=${encodeURIComponent(this.token)}`;
    this.ws = new WS(fullUrl) as WebSocket;

    this.ws.onmessage = (event) => {
      let msg: ServerMessage;
      try {
        msg = JSON.parse(event.data as string) as ServerMessage;
      } catch {
        return;
      }
      if (msg.type === "game_state") {
        this.opts.onReady?.(msg);
      }
      this.opts.onMessage?.(msg);
    };

    this.ws.onclose = () => this.opts.onClose?.();
    this.ws.onerror = (err) => this.opts.onError?.(err);
  }

  send(msg: ClientMessage): void {
    if (!this.ws || this.ws.readyState !== this.ws.OPEN) {
      throw new Error("BikClient: not connected");
    }
    this.ws.send(JSON.stringify(msg));
  }

  close(): void {
    this.ws?.close();
  }
}
