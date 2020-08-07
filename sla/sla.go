package sla

import (
	"log"
	"time"

	"github.com/jmartin127/linear-autolabeler/linear"
	"github.com/rickar/cal/v2"
	"github.com/rickar/cal/v2/us"
)

func NewSLA(lc *linear.LinearClient) *SLA {
	return &SLA{
		lc: lc,
	}
}

type SLA struct {
	lc *linear.LinearClient
}

func (s *SLA) ExceedsSLA(issue *linear.IssueNode, loc *time.Location) (bool, time.Duration, time.Duration) {
	if issue.State.Name == "Ready for Review" {
		return exceedsSLAInBusinessHours(issue, loc, "Ready for Review", time.Hour*time.Duration(8))
	} else if issue.State.Name == "Accepted" {
		return exceedsSLAInBusinessHours(issue, loc, "Accepted", time.Hour*time.Duration(16))
	} else if issue.State.Name == "In Progress" {
		return exceedsSLAInBusinessHours(issue, loc, "Accepted", time.Hour*time.Duration(16))
	} else if issue.State.Name == "Verify" {
		return exceedsSLAInBusinessHours(issue, loc, "Verify", time.Hour*time.Duration(8))
	} else if issue.State.Name == "Waiting on Partner" {
		return exceedsSLAInBusinessHours(issue, loc, "Waiting on Partner", time.Hour*time.Duration(80))
	} else if issue.State.Name == "Additional Info Required" {
		exceedsSLA, _, _ := exceedsSLAInBusinessHours(issue, loc, "Additional Info Required", time.Hour*time.Duration(16))
		if exceedsSLA {
			lastCommentTime, err := s.lc.GetLastTimeIssueWasCommentedOn(issue)
			if err != nil {
				log.Fatal(err) // TODO
			}
			exceedsSLAForCommentToBeAdded, timeOverdueForComment, sla := exceedsSLAInBusinessHoursForStart(lastCommentTime, loc, time.Hour*time.Duration(16))
			if exceedsSLAForCommentToBeAdded {
				return exceedsSLAForCommentToBeAdded, timeOverdueForComment, sla
			}
			return false, time.Hour, time.Hour
		}
		return false, time.Hour, time.Hour
	}

	return false, time.Hour, time.Hour
}

func exceedsSLAInBusinessHours(issue *linear.IssueNode, loc *time.Location, refState string, sla time.Duration) (bool, time.Duration, time.Duration) {
	timeEnteredCurrentState := linear.GetLastTimeIssueEnteredState(issue, refState)
	return exceedsSLAInBusinessHoursForStart(timeEnteredCurrentState, loc, sla)
}

func exceedsSLAInBusinessHoursForStart(refTime time.Time, loc *time.Location, sla time.Duration) (bool, time.Duration, time.Duration) {
	start := refTime.In(loc)
	end := time.Now().In(loc)
	durationInCurrentStateBusinessHours := businessDurationBetweenTimes(start, end)

	if durationInCurrentStateBusinessHours > sla {
		// determine how much the SLA is exceeded
		exceedsSLABySeconds := durationInCurrentStateBusinessHours.Seconds() - sla.Seconds()
		return true, (time.Second * time.Duration(exceedsSLABySeconds)), sla
	}

	return false, time.Hour, time.Hour
}

func businessDurationBetweenTimes(start, end time.Time) time.Duration {
	c := cal.NewBusinessCalendar()

	// add holidays that the business observes
	c.AddHoliday(
		us.NewYear,
		us.MemorialDay,
		us.IndependenceDay,
		us.LaborDay,
		us.ThanksgivingDay,
		us.ChristmasDay,
	)

	return c.WorkHoursInRange(start, end)
}
