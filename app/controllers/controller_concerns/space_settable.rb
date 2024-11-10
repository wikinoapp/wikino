# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  module SpaceSettable
    extend T::Sig
    extend ActiveSupport::Concern

    sig(:final) { void }
    def set_current_space
      Current.space = Space.kept.find_by!(identifier: request.subdomain)
    end
  end
end
