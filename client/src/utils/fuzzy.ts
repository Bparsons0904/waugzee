import type { Release } from "@models/Release";
import type { UserRelease } from "@models/User";
import Fuse from "fuse.js";

const defaultOptions = {
  keys: [
    { name: "artists.artist.name", weight: 2.0 },
    { name: "title", weight: 1.8 },
    { name: "genres.name", weight: 1 },
    { name: "labels.label.name", weight: 0.8 },
  ],
  isCaseSensitive: false,
  includeScore: true,
  shouldSort: true,
  threshold: 0.4,
  distance: 100,
  minMatchCharLength: 2,
  ignoreLocation: true,
  useExtendedSearch: false,
  findAllMatches: true,
};

export const createFuseInstance = (releases: Release[], options = {}) => {
  return new Fuse(releases, { ...defaultOptions, ...options });
};

export const fuzzySearchReleases = (
  releases: Release[],
  searchTerm: string,
  options = {},
): Release[] => {
  if (!searchTerm.trim()) {
    return releases;
  }

  const fuse = createFuseInstance(releases, options);
  const results = fuse.search(searchTerm);

  return results.map((result) => result.item);
};

export const customSearchReleases = (
  releases: Release[],
  searchTerm: string,
  filterFn?: (release: Release) => boolean,
): Release[] => {
  const filteredReleases = filterFn ? releases.filter(filterFn) : releases;

  if (!searchTerm.trim()) {
    return filteredReleases;
  }

  return fuzzySearchReleases(filteredReleases, searchTerm);
};

const userReleaseOptions = {
  keys: [
    { name: "release.artists.name", weight: 3.0 },
    { name: "release.title", weight: 2.0 },
    { name: "release.genres.name", weight: 1.0 },
  ],
  isCaseSensitive: false,
  includeScore: true,
  shouldSort: true,
  threshold: 0.4,
  distance: 100,
  minMatchCharLength: 2,
  ignoreLocation: true,
  useExtendedSearch: false,
  findAllMatches: true,
};

export const createUserReleaseFuseInstance = (userReleases: UserRelease[], options = {}) => {
  return new Fuse(userReleases, { ...userReleaseOptions, ...options });
};

export const fuzzySearchUserReleases = (
  userReleases: UserRelease[],
  searchTerm: string,
  options = {},
): UserRelease[] => {
  if (!searchTerm.trim()) {
    return userReleases;
  }

  const fuse = createUserReleaseFuseInstance(userReleases, options);
  const results = fuse.search(searchTerm);

  return results.map((result) => result.item);
};
