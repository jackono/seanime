import { useServerMutation } from "@/api/client/requests"
import { SetTrackerMode_Variables } from "@/api/generated/endpoint.types"
import { API_ENDPOINTS } from "@/api/generated/endpoints"
import { Status } from "@/api/generated/types"
import { useSetServerStatus } from "@/app/(main)/_hooks/use-server-status"
import { useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

export function useSetTrackerMode() {
    const queryClient = useQueryClient()
    const setServerStatus = useSetServerStatus()

    return useServerMutation<Status, SetTrackerMode_Variables>({
        endpoint: API_ENDPOINTS.TRACKER.SetTrackerMode.endpoint,
        method: API_ENDPOINTS.TRACKER.SetTrackerMode.methods[0],
        mutationKey: [API_ENDPOINTS.TRACKER.SetTrackerMode.key],
        onSuccess: async data => {
            if (data) {
                setServerStatus(data)
            }
            await queryClient.invalidateQueries({ queryKey: [API_ENDPOINTS.ANIME_COLLECTION.GetLibraryCollection.key] })
            await queryClient.invalidateQueries({ queryKey: [API_ENDPOINTS.ANILIST.GetAnimeCollection.key] })
            await queryClient.invalidateQueries({ queryKey: [API_ENDPOINTS.ANILIST.GetRawAnimeCollection.key] })
            toast.success("Tracking service updated")
        },
        onError: async error => {
            toast.error(error.message)
        },
    })
}
