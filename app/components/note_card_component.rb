# frozen_string_literal: true

class NoteCardComponent < ApplicationComponent
  def initialize(note_entity:)
    @note_entity = note_entity
  end

  private

  attr_reader :note_entity

  def card_title_color
    note_entity.cover_image_url.present? ? "text-white" : "text-body"
  end
end
