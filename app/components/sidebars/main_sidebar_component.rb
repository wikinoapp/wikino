# typed: strict
# frozen_string_literal: true

module Sidebars
  class MainSidebarComponent < ApplicationComponent
    sig { params(joined_lists: List::PrivateRelation).void }
    def initialize(joined_lists:)
      @joined_lists = joined_lists
    end

    sig { returns(List::PrivateRelation) }
    attr_reader :joined_lists
    private :joined_lists
  end
end
