# typed: strict
# frozen_string_literal: true

module Views
  module Components
    module Cards
      class Page < VC::Base
        sig { params(page: ::Page, show_topic_name: T::Boolean, class_name: String).void }
        def initialize(page:, show_topic_name: true, class_name: "")
          @page = page
          @show_topic_name = show_topic_name
          @class_name = class_name
        end

        sig { returns(::Page) }
        attr_reader :page
        private :page

        sig { returns(T::Boolean) }
        attr_reader :show_topic_name
        private :show_topic_name

        sig { returns(String) }
        attr_reader :class_name
        private :class_name
      end
    end
  end
end
