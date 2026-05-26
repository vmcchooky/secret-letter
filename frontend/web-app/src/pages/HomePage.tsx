import { CreateSecretForm } from "../components/CreateSecretForm";

export function HomePage() {
  return (
    <main className="otl-shell">
      <section className="otl-minimal-header" aria-labelledby="home-title">
        <div className="otl-star-compass" aria-hidden="true">
          <span className="otl-star-ring" />
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.4">
            <path d="M12 2l2.5 7.5L22 12l-7.5 2.5L12 22l-2.5-7.5L2 12l7.5-2.5L12 2Z" fill="currentColor" fillOpacity="0.22" />
          </svg>
        </div>
        <p className="otl-brand">Powered by Quorix</p>
        <h1 id="home-title">
          <span className="otl-title-gradient">Secret Letter</span>
        </h1>
      </section>

      <section className="otl-create-panel">
        <CreateSecretForm />
      </section>
    </main>
  );
}
