# frozen_string_literal: true

class LinksComponent < ApplicationComponent
  def initialize(note_entity:)
    @note_entity = note_entity
    @link_entities = note_entity.links
  end

  private

  attr_reader :link_entities, :note_entity
end
