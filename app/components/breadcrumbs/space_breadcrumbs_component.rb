# typed: strict
# frozen_string_literal: true

module Breadcrumbs
  class SpaceBreadcrumbsComponent < ApplicationComponent
    renders_many :items, Breadcrumbs::SpaceBreadcrumbs::ItemComponent

    sig { params(space: Space).void }
    def initialize(space:)
      @space = space
    end

    sig { returns(Space) }
    attr_reader :space
    private :space
  end
end
