import { Component, Match, Switch } from "solid-js";
import { A } from "@solidjs/router";
import styles from "./NavBar.module.scss";
import { useAuth } from "@context/AuthContext";
import { FRONTEND_ROUTES } from "@constants/api.constants";

export const NavBar: Component = () => {
  const { isAuthenticated, logout } = useAuth();

  return (
    <>
      <nav class={styles.navbar}>
        <div class={styles.navbarContainer}>
          <div class={styles.navbarLogo}>
            <A href={FRONTEND_ROUTES.HOME} class={styles.navbarTitle}>
              Waugzee
            </A>
          </div>
          <div class={styles.navbarMenu}>
            <ul class={styles.navbarItems}>
              <li class={styles.navbarItem}>
                <A
                  href="/"
                  class={styles.navbarLink}
                  activeClass={styles.active}
                  end
                >
                  Home
                </A>
              </li>
              <Switch>
                <Match when={isAuthenticated()}>
                  <li class={styles.navbarItem}>
                    <A
                      href="/workstation"
                      class={styles.navbarLink}
                      activeClass={styles.active}
                    >
                      Workstation
                    </A>
                  </li>
                  <li class={styles.navbarItem}>
                    <A
                      href="/loadtest"
                      class={styles.navbarLink}
                      activeClass={styles.active}
                    >
                      Load Test
                    </A>
                  </li>
                  <li class={styles.navbarItem}>
                    <A
                      href={FRONTEND_ROUTES.PROFILE}
                      class={styles.navbarLink}
                      activeClass={styles.active}
                    >
                      Profile
                    </A>
                  </li>
                  <li class={styles.navbarItem}>
                    <A
                      href={FRONTEND_ROUTES.ADMIN}
                      class={styles.navbarLink}
                      activeClass={styles.active}
                    >
                      Admin
                    </A>
                  </li>
                  {/* <li class={styles.navbarItem}> */}
                  {/*   <A */}
                  {/*     href="/story/1" */}
                  {/*     class={styles.navbarLink} */}
                  {/*     activeClass={styles.active} */}
                  {/*   > */}
                  {/*     Story */}
                  {/*   </A> */}
                  {/* </li> */}
                </Match>
              </Switch>
              <li class={styles.navbarItem}>
                <Switch>
                  <Match when={!isAuthenticated()}>
                    <A
                      href={FRONTEND_ROUTES.LOGIN}
                      class={styles.navbarLink}
                      activeClass={styles.active}
                    >
                      Login
                    </A>
                  </Match>
                  <Match when={isAuthenticated()}>
                    <A href={FRONTEND_ROUTES.HOME} class={styles.navbarLink} onClick={logout}>
                      Logout
                    </A>
                  </Match>
                </Switch>
              </li>
            </ul>
          </div>
        </div>
      </nav>
    </>
  );
};
