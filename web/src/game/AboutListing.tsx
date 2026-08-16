import type { EpilogueView } from "../engine/run";
import { forDisplay } from "../engine/display";
import { useViewTop } from "./useViewTop";

// The transparency panel entry point: the listing as written, next to
// what the game took from it. Reachable only from the reveal, after the
// photo, never before. The record keeps the truth, every listing field
// the game used is on this page, so the promise below is whole.
export function AboutListing({ view, onBack }: { view: EpilogueView; onBack: () => void }) {
  useViewTop();
  const l = view.listing;
  const quotes = l.quotes ?? [];
  // the sanitizer keeps each block of the listing on its own line
  const paragraphs = l.description
    .split("\n")
    .map(forDisplay)
    .filter((p) => p.replace(/[.\s]/g, "") !== "");
  return (
    <section className="about">
      <h2>About {view.name}, from the listing</h2>
      {/* a copy of the page, with the day it was copied. Listings change
          while nobody is looking, and one of the twelve already has, so
          this is a snapshot with a date on it, not the page as it stands */}
      <p className="about-asof">as read on {l.retrieved_on}</p>
      <dl className="about-facts">
        <dt>{view.adopted ? "was listed by" : "listed by"}</dt>
        <dd>
          {view.org_name}, {view.org_city}, {view.org_state}
        </dd>
        <dt>breed</dt>
        <dd>{l.breed}</dd>
        {l.sex && (
          <>
            <dt>sex, as listed</dt>
            <dd>{l.sex}</dd>
          </>
        )}
        {l.age_text && (
          <>
            <dt>age, as written</dt>
            <dd>{l.age_text}</dd>
          </>
        )}
        {l.weight_text && (
          <>
            <dt>weight, as written</dt>
            <dd>{l.weight_text}</dd>
          </>
        )}
        {view.long_stay && (
          <>
            <dt>long stay</dt>
            <dd>
              {l.long_stay_evidence === "placement"
                ? "filed in the long stay section of the listings"
                : "the listing says so"}
            </dd>
          </>
        )}
      </dl>
      <h3>What the game took from the listing</h3>
      {l.default || quotes.length === 0 ? (
        <p className="about-note">
          The game could not read this listing closely this time, so it used only the fields above.
        </p>
      ) : (
        <>
          <p className="about-note">
            The listing's own words, word for word. Everything the game showed you about {view.name} was
            built from the fields above and these words, nothing else.
          </p>
          <ul className="about-quotes">
            {quotes.map((q) => (
              <li key={q}>{forDisplay(q)}</li>
            ))}
          </ul>
        </>
      )}
      <h3>The full listing text</h3>
      <blockquote className="quoted">
        {paragraphs.map((p, i) => (
          <p key={i}>{p}</p>
        ))}
      </blockquote>
      <p className="quoted-source">
        {view.org_name}, {view.org_city}, {view.org_state}
      </p>
      <div className="about-actions">
        <a className="btn btn-primary" href={view.listing_url} target="_blank" rel="noopener noreferrer">
          {view.adopted ? `${view.name}'s page` : `meet ${view.name}`}
        </a>
        <button type="button" className="btn-quiet" onClick={onBack}>
          back
        </button>
      </div>
    </section>
  );
}
