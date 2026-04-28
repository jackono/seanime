import { useMALLogout } from "@/api/hooks/mal.hooks"
import { useServerStatus } from "@/app/(main)/_hooks/use-server-status"
import { Button } from "@/components/ui/button"
import React from "react"
import { BiCheckCircle, BiLogOut } from "react-icons/bi"
import { SiMyanimelist } from "react-icons/si"

const MAL_CLIENT_ID = "51cb4294feb400f3ddc66a30f9b9a00f"

export default function _page() {
    const status = useServerStatus()
    const { mutate: logout, isPending, isSuccess } = useMALLogout()

    const oauthUrl = React.useMemo(() => {
        const challenge = generateRandomString(50)
        const state = generateRandomString(10)
        sessionStorage.setItem("mal-" + state, challenge)
        return `https://myanimelist.net/v1/oauth2/authorize?response_type=code&client_id=${MAL_CLIENT_ID}&state=${state}&code_challenge=${challenge}&code_challenge_method=plain`
    }, [])

    const isLocalhost = window?.location?.host === "127.0.0.1:43211"

    if (!status?.malConnected && !isLocalhost) {
        return (
            <div className="p-12 pt-0 space-y-4 text-center">
                <p className="flex justify-center w-full text-8xl"><SiMyanimelist /></p>
                <h2>Connect MyAnimeList</h2>
                <p className="text-[--muted]">
                    MAL authentication must be started from <em>127.0.0.1:43211</em>.
                </p>
            </div>
        )
    }

    if (!status?.malConnected) {
        return (
            <div className="p-12 pt-0 space-y-6 text-center">
                <p className="flex justify-center w-full text-8xl"><SiMyanimelist /></p>
                <div>
                    <h2>Connect MyAnimeList</h2>
                    <p className="text-[--muted]">Enable MAL progress updates when AniList is unavailable.</p>
                </div>
                <Button onClick={() => window.open(oauthUrl, "_self")} intent="primary" size="lg">
                    Log in with MAL
                </Button>
            </div>
        )
    }

    return (
        <div className="p-12 pt-0 space-y-6">
            <div className="flex items-center justify-between gap-4">
                <p className="flex items-center gap-4 text-6xl">
                    <SiMyanimelist />
                    <span className="text-2xl">MyAnimeList connected</span>
                </p>
                <Button
                    intent="alert-subtle"
                    size="sm"
                    loading={isPending || isSuccess}
                    leftIcon={<BiLogOut />}
                    onClick={() => logout()}
                >
                    Log out
                </Button>
            </div>
            <div className="border rounded-[--radius] p-4 bg-[--paper] text-lg space-y-2">
                <p className="flex items-center gap-2">
                    <BiCheckCircle className="text-green-300" />
                    Progress tracking fallback is ready.
                </p>
                <p className="text-[--muted]">
                    When AniList progress updates fail, Seanime will try to update MAL with the current episode progress.
                </p>
            </div>
        </div>
    )
}

function generateRandomString(length: number): string {
    const possible = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"
    let text = ""
    for (let i = 0; i < length; i++) {
        text += possible.charAt(Math.floor(Math.random() * possible.length))
    }
    return text
}
