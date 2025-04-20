# typed: true
# frozen_string_literal: true

class Topic
  class Policy < ApplicationModel
    sig { returns(T::Boolean) }
    attr_accessor :can_update
    alias_method :can_update?, :can_update

    sig { returns(T::Boolean) }
    attr_accessor :can_destroy
    alias_method :can_destroy?, :can_destroy
  end
end
