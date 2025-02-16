# typed: strict
# frozen_string_literal: true

module Footers
  class PublicComponent < ApplicationComponent
    sig { returns(T::Boolean) }
    def render?
      !Current.viewer.signed_in?
    end
  end
end
