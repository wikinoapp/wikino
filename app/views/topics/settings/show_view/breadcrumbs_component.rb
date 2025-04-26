# typed: strict
# frozen_string_literal: true

module Topics
  module Settings
    class ShowView
      class BreadcrumbsComponent < ApplicationComponent
        sig { params(topic: Topic).void }
        def initialize(topic:)
          @topic = topic
        end

        sig { returns(Topic) }
        attr_reader :topic
        private :topic

        delegate :space, to: :topic
      end
    end
  end
end
