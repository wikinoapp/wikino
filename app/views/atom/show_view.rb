# typed: strict
# frozen_string_literal: true

module Atom
  class ShowView < ApplicationView
    sig { params(space: SpaceRecord, pages: PageRecord::PrivateAssociationRelation).void }
    def initialize(space:, pages:)
      @space = space
      @pages = pages
    end

    sig { returns(SpaceRecord) }
    attr_reader :space
    private :space

    sig { returns(PageRecord::PrivateAssociationRelation) }
    attr_reader :pages
    private :pages

    sig { returns(Integer) }
    def schema_date
      2025
    end

    sig { params(page: PageRecord).returns(String) }
    def entry_id(page:)
      "tag:Wikino,#{schema_date}:Page/#{page.id}"
    end
  end
end
