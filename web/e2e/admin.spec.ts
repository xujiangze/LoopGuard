import { test, expect } from "@playwright/test"

async function loginAsAdmin(page) {
  await page.goto("/#/login")
  await page.evaluate(() => {
    localStorage.removeItem("lg_token")
    localStorage.removeItem("lg_user")
  })
  await page.reload()
  await page.locator("#username").fill("admin")
  await page.locator("#password").fill("admin123")
  await page.locator('button[type="submit"]').click()
  await expect(page.getByRole("heading", { name: "工单列表" })).toBeVisible()
}

test.describe("程序管理", () => {
  test("注册新程序", async ({ page }) => {
    await loginAsAdmin(page)
    await page.getByRole("link", { name: "程序管理" }).click()
    await expect(page.getByRole("heading", { name: "程序管理" })).toBeVisible()

    await page.getByRole("button", { name: "注册新程序" }).click()
    await expect(page.getByRole("heading", { name: "注册新程序" })).toBeVisible()

    // 弹窗中的输入框 - 使用 label 文本定位
    const dialog = page.locator('[role="dialog"]')
    await dialog.locator("input").nth(0).fill("e2e-proj-new")
    await dialog.locator("input").nth(1).fill("test-cli-new")
    await dialog.locator("input").nth(2).fill("/bin/echo")
    await dialog.locator('input[type="number"]').nth(0).fill("1")
    await dialog.locator('input[type="number"]').nth(1).fill("120")

    await page.getByRole("button", { name: "注册" }).click()
    await expect(page.locator("text=e2e-proj-new")).toBeVisible()
  })

  test("编辑程序", async ({ page }) => {
    await loginAsAdmin(page)
    await page.getByRole("link", { name: "程序管理" }).click()
    await expect(page.getByRole("heading", { name: "程序管理" })).toBeVisible()

    const editBtn = page.getByRole("button", { name: "编辑" }).first()
    await expect(editBtn).toBeVisible()
    await editBtn.click()

    await expect(page.getByRole("heading", { name: "编辑程序" })).toBeVisible()
    const timeoutInput = page.locator('[role="dialog"] input[type="number"]').last()
    await timeoutInput.clear()
    await timeoutInput.fill("300")

    await page.getByRole("button", { name: "保存" }).click()
    await expect(page.getByRole("heading", { name: "编辑程序" })).not.toBeVisible()
  })
})

test.describe("用户管理", () => {
  test("创建新用户", async ({ page }) => {
    await loginAsAdmin(page)
    await page.getByRole("link", { name: "用户管理" }).click()
    await expect(page.getByRole("heading", { name: "用户管理" })).toBeVisible()

    await page.getByRole("button", { name: "创建用户" }).click()
    await expect(page.getByRole("heading", { name: "创建用户" })).toBeVisible()

    const dialog = page.locator('[role="dialog"]')
    await dialog.locator("input").nth(0).fill("e2e-new-user")
    await dialog.locator('input[type="password"]').fill("password123")
    await page.getByRole("button", { name: "创建" }).click()

    await expect(page.locator("text=e2e-new-user")).toBeVisible()
  })
})

test.describe("API Key 管理", () => {
  test("创建 API Key 并展示明文", async ({ browser }) => {
    // 需要授权剪贴板权限
    const context = await browser.newContext({
      permissions: ["clipboard-read", "clipboard-write"],
    })
    const page = await context.newPage()

    await page.goto("/#/login")
    await page.evaluate(() => {
      localStorage.removeItem("lg_token")
      localStorage.removeItem("lg_user")
    })
    await page.reload()
    await page.locator("#username").fill("admin")
    await page.locator("#password").fill("admin123")
    await page.locator('button[type="submit"]').click()
    await expect(page.getByRole("heading", { name: "工单列表" })).toBeVisible()

    await page.getByRole("link", { name: "API Key 管理" }).click()
    await expect(page.getByRole("heading", { name: "API Key 管理" })).toBeVisible()

    await page.getByRole("button", { name: "创建 Key" }).click()
    await expect(page.getByRole("heading", { name: "创建 API Key" })).toBeVisible()

    const dialog = page.locator('[role="dialog"]')
    await dialog.locator("input").fill("e2e-test-key")
    await page.getByRole("button", { name: "创建" }).click()

    // 明文 Key 展示弹窗
    await expect(page.locator("text=请立即复制，此密钥只显示一次")).toBeVisible()
    await expect(page.locator("text=lg_")).toBeVisible()

    await page.getByRole("button", { name: "复制到剪贴板" }).click()
    await expect(page.getByRole("button", { name: "已复制" })).toBeVisible()

    await page.getByRole("button", { name: "我已保存，关闭" }).click()
    await expect(page.locator("text=请立即复制")).not.toBeVisible()

    await expect(page.locator("text=e2e-test-key").first()).toBeVisible()

    await context.close()
  })
})
