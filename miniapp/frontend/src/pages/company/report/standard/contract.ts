// Report content contract derived from the user-approved field schema (#559).
// English keys carry report content; rendering and icons remain page-owned.
export type CompanyReport = {
  readonly report_info: {
    readonly company_name: string | null;
    readonly stock_code: string | null;
    readonly report_date: string | null;
  };
  readonly decision: {
    readonly summary: {
      readonly core_conclusion: {
        readonly core_conclusion: string | null;
      };
      readonly direction: {
        readonly direction: string | null;
        readonly direction_note: string | null;
      };
      readonly confidence: {
        readonly confidence: string | null;
        readonly confidence_note: string | null;
      };
      readonly risk_level: {
        readonly risk_level: string | null;
        readonly risk_score_scale: {
          readonly score: number | null;
          readonly max_score: number | null;
        };
      };
      readonly position: {
        readonly position_cap: string | null;
        readonly position_center: string | null;
      };
    };
    readonly detail: {
      readonly decision_factors: ReadonlyArray<{
        readonly factor_title: string | null;
        readonly key_data_summary: string | null;
        readonly evidence_type: string | null;
      }>;
      readonly rejections: ReadonlyArray<{
        readonly reason_title: string | null;
        readonly reason_content: string | null;
      }>;
    };
  };
  readonly four_dimensions: {
    readonly summary: {
      readonly technical: {
        readonly score: number | null;
        readonly rating: string | null;
        readonly core_judgment: string | null;
        readonly key_evidence_summary: string | null;
      };
      readonly fundamental: {
        readonly score: number | null;
        readonly rating: string | null;
        readonly core_judgment: string | null;
        readonly key_evidence_summary: string | null;
      };
      readonly news: {
        readonly score: number | null;
        readonly rating: string | null;
        readonly core_judgment: string | null;
        readonly key_evidence_summary: string | null;
      };
      readonly sentiment: {
        readonly score: number | null;
        readonly rating: string | null;
        readonly core_judgment: string | null;
        readonly key_evidence_summary: string | null;
      };
    };
    readonly detail: {
      readonly technical: {
        readonly assessment: {
          readonly score: number | null;
          readonly rating: string | null;
          readonly score_note: string | null;
          readonly core_conclusion: string | null;
          readonly judgment_note: string | null;
        };
        readonly reasons: ReadonlyArray<{
          readonly reason_title: string | null;
          readonly reason_content: string | null;
        }>;
        readonly key_metrics: ReadonlyArray<{
          readonly metric_name: string | null;
          readonly metric_value: string | null;
          readonly metric_note: string | null;
        }>;
        readonly key_readings: {
          readonly price_and_moving_averages: {
            readonly close_price: number | null;
            readonly reading_basis: string | null;
            readonly moving_averages: ReadonlyArray<{
              readonly average_name: string | null;
              readonly average_value: number | null;
              readonly comparison_note: string | null;
            }>;
          };
          readonly momentum_volatility_volume: {
            readonly analysis_title: string | null;
            readonly metric_groups: ReadonlyArray<{
              readonly group_name: string | null;
              readonly readings: string | null;
              readonly metric_note: string | null;
            }>;
          };
          readonly period_returns: ReadonlyArray<{
            readonly period: string | null;
            readonly return_pct: number | null;
          }>;
        };
        readonly support_and_resistance: {
          readonly price_map: {
            readonly map_title: string | null;
            readonly price_basis: string | null;
            readonly close_price: number | null;
            readonly levels: ReadonlyArray<{
              readonly level_code: string | null;
              readonly level_type: string | null;
              readonly price_value: {
                readonly min: number | null;
                readonly max: number | null;
              };
              readonly price_basis_note: string | null;
            }>;
          };
          readonly risk_boundary_note: {
            readonly note_title: string | null;
            readonly note_content: string | null;
          };
        };
      };
      readonly fundamental: {
        readonly assessment: {
          readonly score: number | null;
          readonly rating: string | null;
          readonly score_note: string | null;
          readonly core_conclusion: string | null;
          readonly conclusion_note: string | null;
        };
        readonly reasons: ReadonlyArray<{
          readonly reason_title: string | null;
          readonly reason_content: string | null;
        }>;
        readonly key_metrics: ReadonlyArray<{
          readonly metric_name: string | null;
          readonly metric_value: string | null;
          readonly metric_note: string | null;
        }>;
        readonly six_dimension_scores: {
          readonly quality_profile: ReadonlyArray<{
            readonly dimension_name: string;
            readonly score: number | null;
          }>;
          readonly margin_expense_comparison: {
            readonly analysis_title: string | null;
            readonly gross_margin_change_pp: number | null;
            readonly expense_ratio_change_pp: number | null;
            readonly difference_note: string | null;
            readonly net_margin_pct: number | null;
            readonly net_margin_change_pp: number | null;
            readonly expense_ratio_pct: number | null;
            readonly prior_expense_ratio_pct: number | null;
          };
          readonly roe_trend: {
            readonly trend_title: string | null;
            readonly basis_note: string | null;
            readonly history: ReadonlyArray<{
              readonly period: string | null;
              readonly roe_pct: number | null;
            }>;
          };
        };
        readonly financial_readings: {
          readonly growth_sources: {
            readonly analysis_title: string | null;
            readonly period: string | null;
            readonly business_highlights: string | null;
            readonly regional_revenue: ReadonlyArray<{
              readonly region_name: string | null;
              readonly revenue_100m_cny: number | null;
              readonly revenue_share_pct: number | null;
              readonly revenue_yoy_pct: number | null;
            }>;
          };
          readonly profit_realization_threshold: {
            readonly analysis_title: string | null;
            readonly calculation_basis: string | null;
            readonly actual_metric_name: string | null;
            readonly actual_metric_value: string | null;
            readonly actual_period_note: string | null;
            readonly required_metric_name: string | null;
            readonly required_metric_value: string | null;
            readonly required_period_note: string | null;
            readonly calculation_note: string | null;
          };
          readonly financial_evidence: ReadonlyArray<{
            readonly category_name: string | null;
            readonly financial_content: string | null;
          }>;
        };
      };
      readonly news: {
        readonly assessment: {
          readonly score: number | null;
          readonly rating: string | null;
          readonly score_note: string | null;
          readonly core_conclusion: string | null;
          readonly conclusion_note: string | null;
        };
        readonly reasons: ReadonlyArray<{
          readonly reason_title: string | null;
          readonly reason_content: string | null;
        }>;
        readonly event_timeline: {
          readonly messages: {
            readonly time_range_note: string | null;
            readonly events: ReadonlyArray<{
              readonly event_date: string | null;
              readonly event_type: string | null;
              readonly event_title: string | null;
              readonly event_content: string | null;
            }>;
          };
        };
        readonly business_progress: {
          readonly business_boundary: {
            readonly boundary_conclusion: string | null;
            readonly boundary_note: string | null;
          };
          readonly robotics_completed_actions: ReadonlyArray<{
            readonly action_date: string | null;
            readonly action_name: string | null;
            readonly action_content: string | null;
          }>;
        };
      };
      readonly sentiment: {
        readonly assessment: {
          readonly score: number | null;
          readonly rating: string | null;
          readonly score_note: string | null;
          readonly core_conclusion: string | null;
          readonly conclusion_note: string | null;
        };
        readonly reasons: ReadonlyArray<{
          readonly reason_title: string | null;
          readonly reason_content: string | null;
        }>;
        readonly key_metrics: ReadonlyArray<{
          readonly metric_name: string | null;
          readonly metric_value: string | null;
          readonly metric_note: string | null;
        }>;
        readonly daily_main_fund_flow: {
          readonly period: string | null;
          readonly unit: string;
          readonly daily_values: ReadonlyArray<{
            readonly trade_date: string | null;
            readonly net_flow: number | null;
          }>;
        };
        readonly key_readings: ReadonlyArray<{
          readonly item_name: string;
          readonly reading_basis?: string | null;
          readonly reading_content: string | null;
        }>;
      };
    };
  };
  readonly debate: {
    readonly summary: {
      readonly final_direction: string | null;
      readonly decision_summary: string | null;
      readonly bull_score: {
        readonly score: number | null;
        readonly max_score: number | null;
      };
      readonly bear_score: {
        readonly score: number | null;
        readonly max_score: number | null;
      };
    };
    readonly detail: {
      readonly assessment: {
        readonly final_direction: string | null;
        readonly decision_date: string | null;
        readonly core_conclusion: string | null;
        readonly conclusion_note: string | null;
      };
      readonly reasons: ReadonlyArray<{
        readonly reason_title: string | null;
        readonly reason_content: string | null;
      }>;
      readonly bear_arguments: ReadonlyArray<{
        readonly argument_title: string | null;
        readonly evidence_category: string | null;
        readonly evidence: string | null;
        readonly supporting_reason: string | null;
      }>;
      readonly bull_arguments: ReadonlyArray<{
        readonly argument_title: string | null;
        readonly evidence_category: string | null;
        readonly evidence: string | null;
        readonly supporting_reason: string | null;
      }>;
    };
  };
  readonly risk: {
    readonly summary: {
      readonly core_conclusion: string | null;
      readonly risk_assessment: {
        readonly score: number | null;
        readonly max_score: number | null;
        readonly level: string | null;
      };
      readonly assessment_basis: string | null;
      readonly mandate: string | null;
      readonly mandate_note: string | null;
      readonly position_requirements: string | null;
      readonly action_timing: string | null;
      readonly timing_note: string | null;
    };
    readonly detail: {
      readonly assessment: {
        readonly risk_score: number | null;
        readonly risk_level: string | null;
        readonly assessment_horizon: string | null;
        readonly core_conclusion: string | null;
        readonly conclusion_note: string | null;
      };
      readonly reasons: ReadonlyArray<{
        readonly reason_title: string | null;
        readonly risk_fact: string | null;
        readonly impact_mechanism: string | null;
        readonly response_requirement: string | null;
      }>;
      readonly risk_inventory: {
        readonly market_risks: ReadonlyArray<{
          readonly risk_title: string | null;
          readonly risk_content: string | null;
        }>;
        readonly company_risks: ReadonlyArray<{
          readonly risk_title: string | null;
          readonly risk_content: string | null;
        }>;
        readonly macro_risks: ReadonlyArray<{
          readonly risk_title: string | null;
          readonly risk_content: string | null;
        }>;
      };
    };
  };
};
