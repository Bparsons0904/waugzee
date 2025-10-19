import { NavBar } from "@layout/Navbar/Navbar";
import type { Component, JSX } from "solid-js";

interface LayoutProps {
  children: JSX.Element;
}

export const Layout: Component<LayoutProps> = (props) => {
  return (
    <>
      <NavBar />
      <main class="content">{props.children}</main>
    </>
  );
};
