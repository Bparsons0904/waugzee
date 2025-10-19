import { Button } from "@components/common/ui/Button/Button";
import { Image } from "@components/common/ui/Image/Image";
import { ROUTES } from "@constants/api.constants";
import { A } from "@solidjs/router";
import type { Component } from "solid-js";
import styles from "./LandingPage.module.scss";

const LandingPage: Component = () => {
  const featureCards = [
    {
      title: "Collection Management",
      description:
        "Organize and track your vinyl records with automatic Discogs integration for complete metadata and artwork.",
      image: "/images/features/collection-management.svg",
      placeholder: "♫♪♫",
    },
    {
      title: "Play Tracking",
      description:
        "Log every listening session with equipment details, duration tracking, and personal notes.",
      image: "/images/features/play-tracking.svg",
      placeholder: "⏯️",
    },
    {
      title: "Equipment Management",
      description:
        "Track your turntables, cartridges, and styluses with wear monitoring and maintenance schedules.",
      image: "/images/features/equipment-management.svg",
      placeholder: "🎧",
    },
    {
      title: "Analytics & Insights",
      description:
        "Discover listening patterns, favorite genres, and collection statistics with beautiful visualizations.",
      image: "/images/features/analytics.svg",
      placeholder: "📊",
    },
  ];

  return (
    <div class={styles.landingPage}>
      <section class={styles.hero}>
        <div class={styles.container}>
          <h1 class={styles.heroTitle}>Welcome to Waugzee</h1>
          <p class={styles.heroSubtitle}>
            Your personal vinyl collection management system. Track your records, log listening
            sessions, and maintain your equipment with ease.
          </p>
          <div class={styles.heroCta}>
            <A href={ROUTES.LOGIN} class={styles.btnLink}>
              <Button variant="gradient" size="lg">
                Start Managing Your Collection
              </Button>
            </A>
          </div>
          <div class={styles.heroImage}>
            <Image
              src="/images/black-white-player.jpg"
              fallback="/images/placeholders/turntable-placeholder.svg"
              alt="Turntable with vinyl record"
              showSkeleton={true}
              loading="eager"
              className={styles.heroImageContent}
            />
          </div>
        </div>
      </section>

      <section class={styles.socialFun}>
        <div class={styles.container}>
          <h2 class={styles.sectionTitle}>Everything You Need for Vinyl Collection Management</h2>
          <div class={styles.socialGrid}>
            {featureCards.map((card) => (
              <div class={styles.socialCard}>
                <div class={styles.socialCardImage}>
                  <Image
                    src={card.image}
                    alt={card.title}
                    aspectRatio="wide"
                    showSkeleton={true}
                    fallback={`data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 200 120"><rect width="200" height="120" fill="%23f3f4f6"/><text x="100" y="60" font-size="36" text-anchor="middle" fill="%236b7280">${card.placeholder}</text></svg>`}
                  />
                </div>
                <h3 class={styles.socialCardTitle}>{card.title}</h3>
                <p class={styles.socialCardDescription}>{card.description}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section class={styles.footerCta}>
        <div class={styles.container}>
          <h2 class={styles.footerTitle}>Ready to Organize Your Collection?</h2>
          <p class={styles.footerSubtitle}>
            Start tracking your vinyl records today and never lose track of your music again.
          </p>
          <A href={ROUTES.LOGIN} class={styles.btnLink}>
            <Button variant="gradient" size="lg">
              Start Your Collection Journey
            </Button>
          </A>
        </div>
      </section>
    </div>
  );
};

export default LandingPage;
