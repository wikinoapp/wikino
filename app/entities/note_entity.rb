# frozen_string_literal: true

class NoteEntity < ApplicationEntity
  attribute? :id, Types::String
  attribute? :database_id, Types::String
  attribute? :title, Types::String
  attribute? :body, Types::String
  attribute? :updated_at, Types::Params::Time

  def self.from_node(note_node)
    attrs = {}

    if id = note_node["id"]
      attrs[:id] = id
    end

    if database_id = note_node["databaseId"]
      attrs[:database_id] = database_id
    end

    if title = note_node["title"]
      attrs[:title] = title
    end

    if body = note_node["body"]
      attrs[:body] = body
    end

    if updated_at = note_node["updatedAt"]
      attrs[:updated_at] = updated_at
    end

    new attrs
  end
end
