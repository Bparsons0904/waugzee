import { useAuth } from "@context/AuthContext";
import Home from "@pages/Home/Home";
import { type Component, Show } from "solid-js";

const Landing: Component = () => {
  const { isAuthenticated } = useAuth();

  return (
    <Show when={isAuthenticated()} fallback={<Home />}>
      <Home />
    </Show>
  );
};

export default Landing;
