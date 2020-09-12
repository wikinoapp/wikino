# frozen_string_literal: true

class NoteEntity < ApplicationEntity
  attribute? :id, Types::String
  attribute? :database_id, Types::String
  attribute? :title, Types::String
  attribute? :body, Types::String
  attribute? :body_html, Types::String
  attribute? :cover_image_url, Types::String
  attribute? :updated_at, Types::Params::Time
  attribute? :links, Types::Array.of(LinkEntity)
  attribute? :backlinks, Types::Array.of(LinkEntity)

  def self.from_node(node)
    attrs = {}

    if id = node["id"]
      attrs[:id] = id
    end

    if database_id = node["databaseId"]
      attrs[:database_id] = database_id
    end

    if title = node["title"]
      attrs[:title] = title
    end

    if body = node["body"]
      attrs[:body] = body
    end

    if body_html = node["bodyHtml"]
      attrs[:body_html] = body_html
    end

    if cover_image_url = node["coverImageUrl"]
      attrs[:cover_image_url] = cover_image_url
    end

    if updated_at = node["updatedAt"]
      attrs[:updated_at] = updated_at
    end

    link_nodes = node.dig("links", "nodes")
    attrs[:links] = LinkEntity.from_nodes(link_nodes || [])

    backlink_nodes = node.dig("backlinks", "nodes")
    attrs[:backlinks] = LinkEntity.from_nodes(backlink_nodes || [])

    new attrs
  end
end
