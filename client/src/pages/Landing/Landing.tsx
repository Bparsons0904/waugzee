import { Component, Show } from "solid-js";
import { useAuth } from "@context/AuthContext";
import Home from "@pages/Home/Home";

const Landing: Component = () => {
  const { isAuthenticated } = useAuth();

  return (
    <Show when={isAuthenticated()} fallback={<Home />}>
      <Home />
    </Show>
  );
};

export default Landing;
