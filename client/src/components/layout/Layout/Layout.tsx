import { SubNavbar } from "@components/SubNavbar/SubNavbar";
import { useAuth } from "@context/AuthContext";
import { NavBar } from "@layout/Navbar/Navbar";
import { type Component, Show } from "solid-js";

export const Layout: Component = () => {
  const { isAuthenticated } = useAuth();

  return (
    <>
      <NavBar />
      <Show when={isAuthenticated()}>
        <SubNavbar />
      </Show>
    </>
  );
};
