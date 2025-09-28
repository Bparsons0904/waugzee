import "./App.scss";
import { Component } from "solid-js";
import { AuthProvider } from "./context/AuthContext";
import { UserDataProvider } from "./context/UserDataContext";
import { QueryClient, QueryClientProvider } from "@tanstack/solid-query";
import { WebSocketProvider } from "@context/WebSocketContext";
import { ToastProvider } from "./context/ToastContext";
import { RouteSectionProps } from "@solidjs/router";
import { NavBar } from "@components/layout/Navbar/Navbar";
import { ProxyService } from "@components/ProxyService";

const App: Component<RouteSectionProps<unknown>> = (props) => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  return (
    <QueryClientProvider client={queryClient}>
      <ToastProvider>
        <AuthProvider>
          <UserDataProvider>
            <WebSocketProvider>
              <ProxyService />
              <NavBar />
              <main class="content">{props.children}</main>
            </WebSocketProvider>
          </UserDataProvider>
        </AuthProvider>
      </ToastProvider>
    </QueryClientProvider>
  );
};

export default App;
