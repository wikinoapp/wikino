# typed: strict
# frozen_string_literal: true

module Footers
  class GlobalComponent < ApplicationComponent
    sig { params(signed_in: T::Boolean, class_name: String).void }
    def initialize(signed_in:, class_name: "")
      @signed_in = signed_in
      @class_name = class_name
    end

    sig { returns(T::Boolean) }
    attr_reader :signed_in
    private :signed_in
    alias_method :signed_in?, :signed_in

    sig { returns(String) }
    attr_reader :class_name
    private :class_name
  end
end
