# typed: strict
# frozen_string_literal: true

class EditorComponent < ApplicationComponent
  sig { params(id: String).void }
  def initialize(id:)
    @id = id
  end

  private

  sig { returns(String) }
  attr_reader :id
end
