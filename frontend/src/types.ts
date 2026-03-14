// API response types (match Go backend / TMDB)

export interface User {
  id: number
  email: string
  created_at: string
  login_count: number
  last_login_at: string | null
}

export interface TVShow {
  id: number
  name: string
  overview: string
  first_air_date: string
  poster_path: string | null
  backdrop_path: string | null
}

export interface TVShowDetails extends TVShow {
  number_of_seasons: number
  seasons: Season[]
  created_by: Person[]
}

export interface Season {
  id: number
  name: string
  overview: string
  season_number: number
  poster_path: string | null
  episodes?: Episode[]
}

export interface Episode {
  id: number
  name: string
  overview: string
  air_date: string
  episode_number: number
  season_number: number
  still_path: string | null
  crew?: Person[]
}

export interface Person {
  id: number
  name: string
  job?: string
  character?: string
  department?: string
  known_for_department?: string
  profile_path: string | null
  credit_id?: string
}

export interface Credit {
  id: number
  credit_id: string
  name: string
  job: string
  department: string
  episode_count: number
  first_air_date: string
  poster_path: string | null
  overview?: string
}

export interface WriterCredit extends Credit {
  episodes: Episode[]
}

export interface AggregateCredit {
  id: number
  name: string
  department: string
  jobs: { credit_id: string; job: string; episode_count: number }[]
  profile_path: string | null
}

export interface UserWithFavoriteCount {
  id: number
  email: string
  favorite_writers_count: number
  login_count: number
  last_login_at: string | null
}

export interface OverlapGraphNode {
  id: string
  type: 'writer' | 'show'
  name: string
  poster_path?: string
  profile_path?: string
  writer_count?: number
  first_air_date?: string
  priority?: number
}

export interface OverlapGraphEdge {
  source: string
  target: string
}
