import { expect, test } from '@playwright/test'

test('app loads with sidebar title', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Star Manager' })).toBeVisible()
})
