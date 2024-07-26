# typed: strict
# frozen_string_literal: true

class EditorComponent < ApplicationComponent
  sig { params(id: String, note: Note).void }
  def initialize(id:, note:)
    @id = id
    @note = note
  end

  private

  sig { returns(String) }
  attr_reader :id

  sig { returns(Note) }
  attr_reader :note
end
