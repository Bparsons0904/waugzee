import { Component } from "solid-js";
import { useAuth } from "@context/AuthContext";
import { Button } from "@components/common/ui/Button/Button";

const Profile: Component = () => {
  const { user, logout } = useAuth();

  return (
    <div style={{ padding: "2rem", "max-width": "600px", margin: "0 auto" }}>
      <h1>Profile</h1>
      <div style={{ margin: "2rem 0" }}>
        <h2>User Information</h2>
        <div style={{ margin: "1rem 0" }}>
          <p>
            <strong>Username:</strong> {user()?.displayName || "Not available"}
          </p>
          <p>
            <strong>Name:</strong> {user()?.firstName || ""}{" "}
            {user()?.lastName || ""}
          </p>
          <p>
            <strong>User ID:</strong> {user()?.id || "Not available"}
          </p>
          <p>
            <strong>Last Login:</strong>{" "}
            {user()?.lastLoginAt
              ? new Date(user().lastLoginAt).toLocaleDateString()
              : "Not available"}
          </p>
          <p>
            <strong>Discogs Token:</strong>{" "}
            {user()?.configuration?.discogsToken ? "Connected" : "Not connected"}
          </p>
          <p>
            <strong>Selected Folder:</strong>{" "}
            {user()?.configuration?.selectedFolderId ? `Folder ID ${user()?.configuration?.selectedFolderId}` : "Not selected"}
          </p>
        </div>
      </div>
      <div>
        <Button variant="primary" onClick={logout}>
          Sign Out
        </Button>
      </div>
    </div>
  );
};

export default Profile;

