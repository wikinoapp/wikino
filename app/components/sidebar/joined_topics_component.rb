# typed: strict
# frozen_string_literal: true

module Sidebar
  class JoinedTopicsComponent < ApplicationComponent
    sig { params(variant: T.nilable(Symbol)).void }
    def initialize(variant: nil)
      @variant = T.let(variant || :fixed, Symbol)
    end

    sig { returns(Symbol) }
    attr_reader :variant
    private :variant

    sig { returns(String) }
    private def turbo_frame_id
      "joined-topics-#{variant}"
    end
  end
end
