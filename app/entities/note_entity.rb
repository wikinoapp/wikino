# frozen_string_literal: true

class NoteEntity < ApplicationEntity
  attribute? :number, Types::Integer
  attribute? :title, Types::String
  attribute? :body, Types::String

  def self.from_node(note_node)
    attrs = {}

    if number = note_node["number"]
      attrs[:number] = number.to_i
    end

    if title = note_node["title"]
      attrs[:title] = title
    end

    if body = note_node["body"]
      attrs[:body] = body
    end

    new attrs
  end
end
