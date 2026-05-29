import { test, expect, request as apiRequest } from "@playwright/test"

const API_KEY = "lg_b09bd37529cc118ed60a597294b80b4c2ed98843b55aa6ce"
const BASE = "http://localhost:8080/api/v1"

async function createTicketViaAPI(): Promise<number> {
  const ctx = await apiRequest.newContext()
  const res = await ctx.post(`${BASE}/tickets`, {
    headers: { "X-API-Key": API_KEY, "Content-Type": "application/json" },
    data: { project: "e2e-project", name: "test-program", args: { target: "test-server" } },
  })
  const body = await res.json()
  return body.ticket_id
}

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

test.describe("工单列表与详情", () => {
  test("工单列表页加载正常", async ({ page }) => {
    await loginAsAdmin(page)
    const table = page.locator("table")
    const emptyText = page.locator("text=暂无工单")
    await expect(table.or(emptyText)).toBeVisible()
  })

  test("状态 Tab 切换", async ({ page }) => {
    await loginAsAdmin(page)
    await page.locator('[role="tab"]', { hasText: "待审批" }).click()
    await page.locator('[role="tab"]', { hasText: "已完成" }).click()
    await page.locator('[role="tab"]', { hasText: "全部" }).click()
  })

  test("API 创建工单后列表显示新工单", async ({ page }) => {
    const ticketId = await createTicketViaAPI()
    await loginAsAdmin(page)
    // 在第一列（ID列）中查找
    await expect(page.locator("table tbody tr").first().locator("td").first()).toContainText(String(ticketId))
  })

  test("查看工单详情页", async ({ page }) => {
    const ticketId = await createTicketViaAPI()
    await loginAsAdmin(page)
    await page.goto(`/#/tickets/${ticketId}`)
    // 等待页面加载
    await page.waitForLoadState("networkidle")
    await expect(page.getByText(`工单 #${ticketId}`)).toBeVisible()
    // 验证详情页各个区域存在
    await expect(page.locator("text=基本信息")).toBeVisible()
    await expect(page.locator("text=AI 提交参数")).toBeVisible()
  })

  test("批准工单", async ({ page }) => {
    const ticketId = await createTicketViaAPI()
    await loginAsAdmin(page)
    await page.goto(`/#/tickets/${ticketId}`)
    await page.waitForLoadState("networkidle")
    await expect(page.getByText(`工单 #${ticketId}`)).toBeVisible()

    await expect(page.locator('button:has-text("批准执行")')).toBeVisible()
    await page.locator('button:has-text("批准执行")').click()
    await expect(page.getByRole("heading", { name: "工单列表" })).toBeVisible()
  })

  test("驳回工单并输入原因", async ({ page }) => {
    const ticketId = await createTicketViaAPI()
    await loginAsAdmin(page)
    await page.goto(`/#/tickets/${ticketId}`)
    await page.waitForLoadState("networkidle")
    await expect(page.getByText(`工单 #${ticketId}`)).toBeVisible()

    await page.locator('button:has-text("驳回")').click()
    await expect(page.locator("text=驳回工单")).toBeVisible()
    await page.locator("textarea").fill("E2E 测试驳回原因")
    await page.locator('button:has-text("确认驳回")').click()
    await expect(page.getByRole("heading", { name: "工单列表" })).toBeVisible()
  })
})
