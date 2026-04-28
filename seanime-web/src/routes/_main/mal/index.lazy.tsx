import Page from "@/app/(main)/mal/_page"
import { createLazyFileRoute } from "@tanstack/react-router"

export const Route = createLazyFileRoute("/_main/mal/")({
    component: Page,
})
