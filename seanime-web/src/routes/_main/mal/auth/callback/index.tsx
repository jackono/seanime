import Page from "@/app/(main)/mal/auth/callback/_page"
import { createFileRoute } from "@tanstack/react-router"

export const Route = createFileRoute("/_main/mal/auth/callback/")({
    component: Page,
})
