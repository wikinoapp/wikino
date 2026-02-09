# typed: strict
# frozen_string_literal: true

class AbsoluteTimeComponent < ApplicationComponent
  sig { params(time: ActiveSupport::TimeWithZone, time_zone: String).void }
  def initialize(time:, time_zone: "Asia/Tokyo")
    @time = time
    @time_zone = time_zone
  end

  sig { returns(String) }
  def call
    time.in_time_zone(time_zone).to_fs(:ymdhm).html_safe
  end

  sig { returns(ActiveSupport::TimeWithZone) }
  attr_reader :time
  private :time

  sig { returns(String) }
  attr_reader :time_zone
  private :time_zone
end
