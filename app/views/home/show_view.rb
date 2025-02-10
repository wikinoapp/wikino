# typed: strict
# frozen_string_literal: true

module Home
  class ShowView < ApplicationView
    use_helpers :set_meta_tags

    sig { params(active_spaces: Space::PrivateCollectionProxy).void }
    def initialize(active_spaces:)
      @active_spaces = active_spaces
    end

    sig { returns(Space::PrivateCollectionProxy) }
    attr_reader :active_spaces
    private :active_spaces

    sig { returns(PageName) }
    private def current_page_name
      PageName::Home
    end
  end
end
