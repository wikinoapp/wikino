# typed: strict
# frozen_string_literal: true

module CardLinks
  class SpaceComponent < ApplicationComponent
    sig { params(space: Space, card_class: String).void }
    def initialize(space:, card_class: "")
      @space = space
      @card_class = card_class
    end

    sig { returns(Space) }
    attr_reader :space
    private :space

    sig { returns(String) }
    attr_reader :card_class
    private :card_class

    sig { returns(String) }
    private def build_card_class
      class_names(
        card_class,
        "bg-base-300 duration-200 ease-in-out grid min-h-[80px] transition px-3 py-2",
        "hover:border hover:border-primary"
      )
    end
  end
end
