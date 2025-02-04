# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  module SpaceFindable
    extend T::Sig
    extend ActiveSupport::Concern

    sig(:final) { returns(Space) }
    def find_space_by_identifier!
      Space.kept.find_by!(identifier: params[:space_identifier])
    end
  end
end
