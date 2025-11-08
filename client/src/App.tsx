import "./App.scss";
import { NavBar } from "@components/layout/Navbar/Navbar";
import { ProxyService } from "@components/ProxyService";
import { SyncStatusProvider } from "@context/SyncStatusContext";
import { WebSocketProvider } from "@context/WebSocketContext";
import type { RouteSectionProps } from "@solidjs/router";
import { QueryClient, QueryClientProvider } from "@tanstack/solid-query";
import { type Component, Suspense } from "solid-js";
import { AuthProvider } from "./context/AuthContext";
import { ToastProvider } from "./context/ToastContext";
import { UserDataProvider } from "./context/UserDataContext";

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
              <SyncStatusProvider>
                <ProxyService />
                <NavBar />
                <main class="content">
                  <Suspense fallback={<div />}>{props.children}</Suspense>
                </main>
              </SyncStatusProvider>
            </WebSocketProvider>
          </UserDataProvider>
        </AuthProvider>
      </ToastProvider>
    </QueryClientProvider>
  );
};

export default App;
