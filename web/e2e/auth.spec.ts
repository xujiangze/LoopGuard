import { test, expect } from "@playwright/test"

test.describe("登录流程", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/#/login")
    await page.evaluate(() => {
      localStorage.removeItem("lg_token")
      localStorage.removeItem("lg_user")
    })
    await page.reload()
    await expect(page).toHaveURL(/\/login/)
  })

  test("未登录访问首页跳转到登录页", async ({ page }) => {
    await page.goto("/#/")
    await expect(page).toHaveURL(/\/login/)
    await expect(page.locator("text=LoopGuard 登录")).toBeVisible()
  })

  test("错误密码登录失败显示错误提示", async ({ page }) => {
    await page.locator("#username").fill("admin")
    await page.locator("#password").fill("wrong-password")
    await page.locator('button[type="submit"]').click()
    // 401 被前端拦截器统一处理
    await expect(page.locator(".text-destructive")).toBeVisible()
  })

  test("admin 登录成功跳转到工单列表", async ({ page }) => {
    await page.locator("#username").fill("admin")
    await page.locator("#password").fill("admin123")
    await page.locator('button[type="submit"]').click()
    await expect(page).toHaveURL(/\/$/)
    await expect(page.getByRole("heading", { name: "工单列表" })).toBeVisible()
    await expect(page.getByRole("link", { name: "程序管理" })).toBeVisible()
  })

  test("普通用户登录不显示管理菜单", async ({ page }) => {
    await page.locator("#username").fill("user1")
    await page.locator("#password").fill("user123")
    await page.locator('button[type="submit"]').click()
    await expect(page).toHaveURL(/\/$/)
    await expect(page.getByRole("heading", { name: "工单列表" })).toBeVisible()
    await expect(page.getByRole("link", { name: "程序管理" })).not.toBeVisible()
  })

  test("退出后跳转到登录页", async ({ page }) => {
    await page.locator("#username").fill("admin")
    await page.locator("#password").fill("admin123")
    await page.locator('button[type="submit"]').click()
    await expect(page.getByRole("heading", { name: "工单列表" })).toBeVisible()
    await page.getByRole("button", { name: "退出" }).click()
    await expect(page).toHaveURL(/\/login/)
  })
})
