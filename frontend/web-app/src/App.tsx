import { lazy, Suspense } from "react";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import { HomePage } from "./pages/HomePage";

const RevealPage = lazy(() =>
  import("./pages/RevealPage").then((module) => ({ default: module.RevealPage })),
);

function RevealRouteFallback() {
  return (
    <main className="otl-shell" aria-busy="true">
      <section
        className="otl-create-panel"
        style={{
          display: "grid",
          justifyItems: "center",
          gap: "12px",
          textAlign: "center",
        }}
      >
        <p className="otl-brand">Quorix One-Time Link</p>
        <h1
          style={{
            margin: 0,
            fontFamily: "var(--qx-font-heading, Georgia, serif)",
            fontSize: "2.2rem",
            fontStyle: "italic",
          }}
        >
          Đang mở mật thư...
        </h1>
        <p style={{ margin: 0, color: "var(--otl-muted)" }}>
          Đang tải phần giải mã, xin chờ một chút.
        </p>
      </section>
    </main>
  );
}

export function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route
          path="/s/:secretId"
          element={
            <Suspense fallback={<RevealRouteFallback />}>
              <RevealPage />
            </Suspense>
          }
        />
        <Route
          path="/reveal/:secretId"
          element={
            <Suspense fallback={<RevealRouteFallback />}>
              <RevealPage />
            </Suspense>
          }
        />
        <Route
          path="/:secretId"
          element={
            <Suspense fallback={<RevealRouteFallback />}>
              <RevealPage />
            </Suspense>
          }
        />
      </Routes>
    </BrowserRouter>
  );
}
