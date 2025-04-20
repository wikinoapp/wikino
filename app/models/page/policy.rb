# typed: true
# frozen_string_literal: true

class Page
  class Policy < ApplicationModel
    sig { returns(T::Boolean) }
    attr_accessor :can_trash
    alias_method :can_trash?, :can_trash
  end
end
