/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_PUBLIC_SECRET_ORIGIN?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
