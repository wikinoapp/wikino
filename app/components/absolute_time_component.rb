# typed: strict
# frozen_string_literal: true

class AbsoluteTimeComponent < ApplicationComponent
  sig { params(viewable: UserEntity, time: ActiveSupport::TimeWithZone).void }
  def initialize(viewable:, time:)
    @viewable = viewable
    @time = time
  end

  sig { returns(String) }
  def call
    time.in_time_zone(viewable.time_zone).to_fs(:ymdhm).html_safe
  end

  sig { returns(UserEntity) }
  attr_reader :viewable
  private :viewable

  sig { returns(ActiveSupport::TimeWithZone) }
  attr_reader :time
  private :time
end
