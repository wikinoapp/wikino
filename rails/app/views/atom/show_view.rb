# typed: strict
# frozen_string_literal: true

module Atom
  class ShowView < ApplicationView
    sig { params(space: Space, pages: T::Array[Page]).void }
    def initialize(space:, pages:)
      @space = space
      @pages = pages
    end

    sig { returns(Space) }
    attr_reader :space
    private :space

    sig { returns(T::Array[Page]) }
    attr_reader :pages
    private :pages

    sig { returns(Integer) }
    def schema_date
      2025
    end

    sig { params(page: Page).returns(String) }
    def entry_id(page:)
      "tag:Wikino,#{schema_date}:Page/#{page.database_id}"
    end
  end
end
