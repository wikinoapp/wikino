# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  module SpaceFindable
    extend T::Sig
    extend ActiveSupport::Concern

    sig(:final) { returns(SpaceRecord) }
    def find_space_by_identifier!
      SpaceRecord.kept.find_by!(identifier: params[:space_identifier])
    end
  end
end
