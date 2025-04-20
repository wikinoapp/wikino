# typed: strict
# frozen_string_literal: true

class EditorComponent < ApplicationComponent
  sig { params(id: String, page: Page).void }
  def initialize(id:, page:)
    @id = id
    @page = page
  end

  private

  sig { returns(String) }
  attr_reader :id

  sig { returns(Page) }
  attr_reader :page
end
