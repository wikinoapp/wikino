# typed: strict
# frozen_string_literal: true

module Breadcrumbs
  class SpaceComponent < ApplicationComponent
    renders_many :items, BaseUI::BreadcrumbComponent::Item

    sig { params(space: Space).void }
    def initialize(space:)
      @space = space
    end

    sig { returns(Space) }
    attr_reader :space
    private :space
  end
end
