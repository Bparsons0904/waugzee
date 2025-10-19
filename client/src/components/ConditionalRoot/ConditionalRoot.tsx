import { useAuth } from "@context/AuthContext";
import HomePage from "@pages/Home/Home";
import LandingPage from "@pages/LandingPage/LandingPage";
import { type Component, Match, Switch } from "solid-js";
import styles from "./ConditionalRoot.module.scss";

export const ConditionalRoot: Component = () => {
  const { authState } = useAuth();

  return (
    <Switch>
      <Match when={authState.status === "loading"}>
        <div class={styles.loadingContainer}>
          <div class={styles.spinner} />
          <p class={styles.loadingText}>Loading...</p>
        </div>
      </Match>
      <Match when={authState.status === "authenticated"}>
        <HomePage />
      </Match>
      <Match when={authState.status === "unauthenticated"}>
        <LandingPage />
      </Match>
    </Switch>
  );
};
