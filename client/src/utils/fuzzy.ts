import Fuse from "fuse.js";
import { Release } from "@models/Release";

const defaultOptions = {
  keys: [
    { name: "title", weight: 2 },
    { name: "artists.artist.name", weight: 1.5 },
    { name: "genres.name", weight: 1 },
    { name: "labels.label.name", weight: 0.8 },
  ],
  isCaseSensitive: false,
  includeScore: true,
  shouldSort: true,
  threshold: 0.4,
  distance: 100,
  minMatchCharLength: 2,
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
