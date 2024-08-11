# typed: false
# frozen_string_literal: true

module Nonoto
  module RoutingConstraint
    class SpaceSubdomain
      def matches?(request)
        request.subdomain.present? && reserved_space_identifiers.exclude?(request.subdomain)
      end

      private def reserved_space_identifiers
        Nonoto.config.reserved_space_identifiers
      end
    end
  end
end
