import { afterEach } from "vitest";
import { cleanup } from "@testing-library/react";

class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}

globalThis.ResizeObserver = ResizeObserverStub;
Element.prototype.scrollIntoView = function scrollIntoView() {};

afterEach(() => {
  cleanup();
});
