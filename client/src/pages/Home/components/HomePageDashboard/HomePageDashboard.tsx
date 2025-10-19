import { Button } from "@components/common/ui/Button/Button";
import { ROUTES } from "@constants/api.constants";
import { useUserData } from "@context/UserDataContext";
import { A } from "@solidjs/router";
import { type Component, createSignal, onMount } from "solid-js";
import styles from "./HomePageDashboard.module.scss";

interface DashboardStats {
  totalRecords: number;
  totalPlays: number;
  listeningHours: number;
  favoriteGenre: string;
}

interface RecentActivity {
  id: string;
  type: "play" | "addition" | "maintenance";
  title: string;
  subtitle: string;
  timestamp: string;
}

const HomePageDashboard: Component = () => {
  const { user } = useUserData();

  const [stats, setStats] = createSignal<DashboardStats>({
    totalRecords: 0,
    totalPlays: 0,
    listeningHours: 0,
    favoriteGenre: "Loading...",
  });
  const [recentActivity, setRecentActivity] = createSignal<RecentActivity[]>([]);
  const [isLoading, setIsLoading] = createSignal(true);

  onMount(async () => {
    try {
      await new Promise((resolve) => setTimeout(resolve, 1000));

      setStats({
        totalRecords: 247,
        totalPlays: 1430,
        listeningHours: 89,
        favoriteGenre: "Jazz",
      });

      setRecentActivity([
        {
          id: "1",
          type: "play",
          title: "Kind of Blue",
          subtitle: "Miles Davis • 2h ago",
          timestamp: "2h ago",
        },
        {
          id: "2",
          type: "addition",
          title: "The Dark Side of the Moon",
          subtitle: "Pink Floyd • Added to collection",
          timestamp: "1d ago",
        },
        {
          id: "3",
          type: "maintenance",
          title: "Stylus Cleaning",
          subtitle: "Audio-Technica AT95E • Cleaned",
          timestamp: "3d ago",
        },
      ]);
    } catch (error) {
      console.error("Failed to load dashboard data:", error);
    } finally {
      setIsLoading(false);
    }
  });

  const quickActions = [
    {
      title: "Log Play",
      description: "Record a listening session",
      icon: "🎵",
      href: "/log",
      variant: "primary" as const,
    },
    {
      title: "Add Record",
      description: "Add to your collection",
      icon: "💽",
      href: "/collection/add",
      variant: "secondary" as const,
    },
    {
      title: "View Collection",
      description: "Browse your records",
      icon: "📚",
      href: "/collection",
      variant: "secondary" as const,
    },
    {
      title: "Analytics",
      description: "View listening insights",
      icon: "📊",
      href: "/analytics",
      variant: "secondary" as const,
    },
  ];

  const getActivityIcon = (type: RecentActivity["type"]) => {
    switch (type) {
      case "play":
        return "▶️";
      case "addition":
        return "💽";
      case "maintenance":
        return "🛠️";
      default:
        return "📅";
    }
  };

  return (
    <div class={styles.homePageDashboard}>
      <div class={styles.container}>
        <section class={styles.hero}>
          <h1 class={styles.heroTitle}>Welcome back, {user()?.firstName || "User"}!</h1>
          <p class={styles.heroSubtitle}>Ready to dive into your vinyl collection?</p>
        </section>

        <section class={styles.statsSection}>
          <div class={styles.statsGrid}>
            <div class={styles.statCard}>
              <div class={styles.statIcon}>💽</div>
              <div class={styles.statContent}>
                <h3 class={styles.statNumber}>
                  {isLoading() ? "--" : stats().totalRecords.toLocaleString()}
                </h3>
                <p class={styles.statLabel}>Records</p>
              </div>
            </div>

            <div class={styles.statCard}>
              <div class={styles.statIcon}>▶️</div>
              <div class={styles.statContent}>
                <h3 class={styles.statNumber}>
                  {isLoading() ? "--" : stats().totalPlays.toLocaleString()}
                </h3>
                <p class={styles.statLabel}>Plays</p>
              </div>
            </div>

            <div class={styles.statCard}>
              <div class={styles.statIcon}>⏱️</div>
              <div class={styles.statContent}>
                <h3 class={styles.statNumber}>
                  {isLoading() ? "--h" : `${stats().listeningHours}h`}
                </h3>
                <p class={styles.statLabel}>Hours</p>
              </div>
            </div>

            <div class={styles.statCard}>
              <div class={styles.statIcon}>🎯</div>
              <div class={styles.statContent}>
                <h3 class={styles.statNumber}>{isLoading() ? "--" : stats().favoriteGenre}</h3>
                <p class={styles.statLabel}>Top Genre</p>
              </div>
            </div>
          </div>
        </section>

        <section class={styles.actionsSection}>
          <h2 class={styles.sectionTitle}>Quick Actions</h2>
          <div class={styles.actionsGrid}>
            {quickActions.map((action) => (
              <A href={action.href} class={styles.actionLink}>
                <div class={styles.actionCard}>
                  <div class={styles.actionIcon}>{action.icon}</div>
                  <h3 class={styles.actionTitle}>{action.title}</h3>
                  <p class={styles.actionDescription}>{action.description}</p>
                  <Button variant={action.variant} size="sm">
                    {action.title}
                  </Button>
                </div>
              </A>
            ))}
          </div>
        </section>

        <section class={styles.activitySection}>
          <div class={styles.activityHeader}>
            <h2 class={styles.sectionTitle}>Recent Activity</h2>
            <A href="/history" class={styles.viewAllLink}>
              View All
            </A>
          </div>

          <div class={styles.activityList}>
            {isLoading() ? (
              <div class={styles.activitySkeleton}>
                {[1, 2, 3].map(() => (
                  <div class={styles.skeletonItem}>
                    <div class={styles.skeletonIcon}></div>
                    <div class={styles.skeletonContent}>
                      <div class={styles.skeletonTitle}></div>
                      <div class={styles.skeletonSubtitle}></div>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              recentActivity().map((activity) => (
                <div class={styles.activityItem}>
                  <div class={styles.activityIcon}>{getActivityIcon(activity.type)}</div>
                  <div class={styles.activityContent}>
                    <h4 class={styles.activityTitle}>{activity.title}</h4>
                    <p class={styles.activitySubtitle}>{activity.subtitle}</p>
                  </div>
                  <span class={styles.activityTime}>{activity.timestamp}</span>
                </div>
              ))
            )}
          </div>
        </section>

        <section class={styles.ctaSection}>
          <div class={styles.ctaCard}>
            <h2 class={styles.ctaTitle}>Explore Your Collection</h2>
            <p class={styles.ctaDescription}>
              Discover new insights about your listening habits and collection patterns.
            </p>
            <A href={ROUTES.DASHBOARD} class={styles.ctaLink}>
              <Button variant="gradient" size="lg">
                Go to Full Dashboard
              </Button>
            </A>
          </div>
        </section>
      </div>
    </div>
  );
};

export default HomePageDashboard;
