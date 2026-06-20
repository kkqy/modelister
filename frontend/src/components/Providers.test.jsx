import React from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import Providers from "./Providers.jsx";
import { api } from "../api.js";

vi.mock("../api.js", async () => {
  const actual = await vi.importActual("../api.js");
  return {
    ...actual,
    api: {
      listProviders: vi.fn(),
      createProvider: vi.fn(),
      listKeys: vi.fn(),
      createKey: vi.fn(),
      updateProvider: vi.fn(),
      deleteProvider: vi.fn(),
      updateKey: vi.fn(),
      deleteKey: vi.fn(),
      syncKey: vi.fn(),
      syncProvider: vi.fn(),
      syncAll: vi.fn(),
    },
  };
});

function deferred() {
  let resolve;
  const promise = new Promise((r) => {
    resolve = r;
  });
  return { promise, resolve };
}

describe("Providers", () => {
  const toast = {
    success: vi.fn(),
    error: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("创建供应商后，在列表刷新完成前也能继续添加 Key", async () => {
    const user = userEvent.setup();
    const pendingReload = deferred();

    api.listProviders
      .mockResolvedValueOnce({ providers: [] })
      .mockReturnValueOnce(pendingReload.promise);
    api.createProvider.mockResolvedValue({
      id: 1,
      name: "OpenAI",
      base_url: "https://api.openai.com",
      note: "",
      enabled: true,
    });
    api.listKeys.mockResolvedValue({ keys: [] });

    render(<Providers toast={toast} onUnauthorized={vi.fn()} />);

    await screen.findByText("还没有供应商，点击「新建供应商」开始。");

    await user.click(screen.getByRole("button", { name: "+ 新建供应商" }));
    await user.type(screen.getByLabelText("名称"), "OpenAI");
    await user.type(screen.getByLabelText("Base URL"), "https://api.openai.com");
    await user.click(screen.getByRole("button", { name: "保存" }));

    expect(await screen.findByText("OpenAI")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /OpenAI/ }));
    await user.click(screen.getByRole("button", { name: "+ 添加 Key" }));

    expect(screen.getByLabelText("API Key")).toBeInTheDocument();

    pendingReload.resolve({
      providers: [
        {
          id: 1,
          name: "OpenAI",
          base_url: "https://api.openai.com",
          note: "",
          enabled: true,
        },
      ],
    });

    await waitFor(() => {
      expect(screen.getByLabelText("API Key")).toBeInTheDocument();
    });
  });
});
