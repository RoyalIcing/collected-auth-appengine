package main

import (
	"context"
	"fmt"

	graphql "github.com/graph-gophers/graphql-go"
)

const schemaString = `
schema {
	query: Query
}

interface Node {
	id: ID!
}

interface Actor {
	#username: String!
}


enum MediaBaseType {
  TEXT
  IMAGE
  AUDIO
  VIDEO
  APPLICATION
}

# See https://www.iana.org/assignments/media-types/media-types.xhtml
type MediaType {
  baseType: String!
  subtype: String!
  parameters: [String]
}

interface Asset {
  mediaType: MediaType!
}

type AssetReference implements Node {
  id: ID!

  asset: Asset
}

type MarkdownDocument implements Asset {
  mediaType: MediaType!
  source: String

  #assetReferences: [AssetReference]
}



type PostsConnection {
	edges: [PostEdge]
	totalCount: Int
}

type PostEdge {
  node: Post
  cursor: ID!
}

type Post implements Node {
  id: ID!

  content: MarkdownDocument
  author: Actor
  #title: String
  #createdAt: UTCTime
	#updatedAt: UTCTime
	
	#repliedTo: Post
	#replies: PostsConnection
}

type Channel implements Node {
	id: ID!

	slug: String

	posts: PostsConnection
}


type Query {
	hello: String!
	channel(slug: String): Channel
	#channel(): Channel
	channels(): [Channel]
}
`

// ChannelsArgs is the arguments take by a Channels resolver
type ChannelsArgs struct{}

// ChannelArgs is the arguments take by a Channel resolver
type ChannelArgs struct {
	Slug *string
}

// Resolver is the interface for concrete implementors
type Resolver interface {
	Hello() string
	Channel(ctx context.Context, args ChannelArgs) (*Channel, error)
	Channels(ctx context.Context /*, args ChannelsArgs*/) (*[]*Channel, error)
}

// MakeSchema creates a GraphQL schema
func MakeSchema(resolver Resolver) *graphql.Schema {
	return graphql.MustParseSchema(schemaString, resolver)
}

// DataStoreResolver reads from the file system
type DataStoreResolver struct {
}

// NewDataStoreResolver makes a new source from the local file system
func NewDataStoreResolver() DataStoreResolver {
	return DataStoreResolver{}
}

// Hello resolved
func (r DataStoreResolver) Hello() string { return "Hello, world!" }

// Channel resolved
func (r DataStoreResolver) Channel(ctx context.Context, args ChannelArgs) (*Channel, error) {
	if args.Slug == nil {
		return nil, fmt.Errorf("Must provide slug")
	}

	channel := NewChannel(*args.Slug)
	return channel, nil
}

// Channels resolved
func (r DataStoreResolver) Channels(ctx context.Context /*, args ChannelsArgs*/) (*[]*Channel, error) {
	return nil, fmt.Errorf("Not implemented")
}
