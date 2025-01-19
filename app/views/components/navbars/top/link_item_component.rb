# typed: strict
# frozen_string_literal: true

module Navbars
  module Top
    class LinkItemComponent < ApplicationComponent
      sig { params(path: String, title: String, icon_name: String, class_name: String).void }
      def initialize(path:, title:, icon_name:, class_name: "")
        @path = path
        @title = title
        @icon_name = icon_name
        @class_name = class_name
      end

      sig { returns(String) }
      attr_reader :path
      private :path

      sig { returns(String) }
      attr_reader :title
      private :title

      sig { returns(String) }
      attr_reader :icon_name
      private :icon_name

      sig { returns(String) }
      attr_reader :class_name
      private :class_name
    end
  end
end
