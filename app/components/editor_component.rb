# typed: strict
# frozen_string_literal: true

class EditorComponent < ApplicationComponent
  sig { params(id: String, body: String).void }
  def initialize(id:, body:)
    @id = id
    @body = body
  end

  private

  sig { returns(String) }
  attr_reader :body, :id
end
